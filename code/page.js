import PropTypes from 'prop-types';
import React from 'react';

import Header from './Header';
import ResumeHeader from './ResumeHeader';


/**
 * The page layout component
 */
const Page = ({ title, stylesheet, main, script, _relativeURL, _ID, _pages, _parents, _nav, _globalProp }) => (
	<html lang="en">
	<head>
	    <meta httpEquiv="Content-Type" content="text/html; charset=utf-8" />
		<title>sudorandom - { title }</title>
		<meta name="description" content="" />
		<meta name="keywords" content="" />
		
		<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1" />
		
		<link href='https://fonts.googleapis.com/css?family=Roboto+Mono:400,100,300italic,300,100italic,400italic,500,500italic,700,700italic&amp;subset=latin,cyrillic' rel='stylesheet' />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/glitche-basic.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/glitche-layout.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/ionicons.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/magnific-popup.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/animate.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/site.css`, _ID )} />

		{
			stylesheet != undefined
				? ( <link rel="stylesheet" href={ _relativeURL( `/assets/css/${ stylesheet }.css`, _ID ) } /> )
				: null
		}

		<link rel="stylesheet" href={_relativeURL( `/assets/css/template-colors/green.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/template-dark/dark.css`, _ID )} />
		<link rel="shortcut icon" href={_relativeURL( `/assets/images/favicons/favicon.ico`, _ID )} />
	</head>
	<body>
		<div className="preloader">
			<div className="centrize full-width">
				<div className="vertical-center">
					<div className="pre-inner">
						<div className="load typing-load"><p>loading...</p></div>
						<span className="typed-load"></span>
					</div>
				</div>
			</div>
		</div>
		
		<div className="container bg">
			<header>
				<div className="head-top">
					<a href="#" className="menu-btn"><span></span></a>
					<div className="top-menu">
						<ul>
						{
							Object.keys(_nav["index"])
								.map(
									(page, i) =>(
										<li key={i} className={ page == _ID ? 'active' : null }><a href={_pages[page]._url} className="lnk">{_pages[page].title}</a></li>
									)
								)
						}
						</ul>
					</div>
				</div>
			</header>

			<div className="wrapper">
				{_ID == 'cv'
			        ? <ResumeHeader title={title} />
			        : <Header title={title} _parents={_parents} _pages={_pages} _globalProp={_globalProp} />
			      }
				<div className="section works">
					<div className="content">
						{ main }
					</div>
				</div>
			</div>

			{
				script != undefined
					? ( <script src={ _relativeURL( `/assets/js/${ script }.js`, _ID ) } /> )
					: null
			}
			
			<footer>
				<div className="soc">
					<a target="_blank" href="https://twitter.com/sudorandom"><span className="ion ion-social-twitter"></span></a>
					<a target="_blank" href="https://github.com/sudorandom"><span className="ion ion-social-github"></span></a>
				</div>
				<div className="copy">© 2022 Kevin McDonald. All rights reserved.</div>
				<div className="clr"></div>
			</footer>
			
			<div className="line top"></div>
			<div className="line bottom"></div>
			<div className="line left"></div>
			<div className="line right"></div>
			
		</div>
		
	    <script src={_relativeURL( `/assets/js/jquery.min.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/jquery.validate.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/typed.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/magnific-popup.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/imagesloaded.pkgd.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/isotope.pkgd.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/glitche-scripts.js`, _ID )}></script>
	</body>
	</html>
);

Page.propTypes = {
/**
	 * title: Homepage
	 */
	title: PropTypes.string.isRequired,

	/**
	 * main: (partials)(5)
	 */
	main: PropTypes.node.isRequired,
};

Page.defaultProps = {};

export default Page;
