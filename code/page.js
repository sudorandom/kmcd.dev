import PropTypes from 'prop-types';
import React from 'react';

import Header from './partials/header';
import ResumeHeader from './partials/resumeheader';
import IndexHeader from './partials/indexheader';


/**
 * The page layout component
 */
const Page = ({ title, stylesheet, main, script, _relativeURL, _ID, _pages, _parents, _nav, _globalProp }) => (
	<html lang="en">
	<head>
	    <meta httpEquiv="Content-Type" content="text/html; charset=utf-8" />
		<title>{`sudorandom - ` + title }</title>
		<meta name="description" content="" />
		<meta name="keywords" content="" />
		<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1" />
		
		<link href='https://fonts.googleapis.com/css?family=Roboto+Mono:400,100,300italic,300,100italic,400italic,500,500italic,700,700italic&amp;subset=latin,cyrillic' rel='stylesheet' />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/glitche-basic.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/glitche-layout.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/animate.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/site.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/magnific-popup.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/js/jqphotoswipe/photoswipe.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/js/jqphotoswipe/default-skin/default-skin.css`, _ID )} />
		
		{
			stylesheet != undefined
				? ( <link rel="stylesheet" href={ _relativeURL( `/assets/css/${ stylesheet }.css`, _ID ) } /> )
				: null
		}

		<link rel="stylesheet" href={_relativeURL( `/assets/css/template-colors/green.css`, _ID )} />
		<link rel="stylesheet" href={_relativeURL( `/assets/css/template-dark/dark.css`, _ID )} />
		<link rel="apple-touch-icon" sizes="180x180" href={_relativeURL( `/assets/images/favicons/apple-touch-icon.png`, _ID )} />
		<link rel="icon" type="image/png" sizes="32x32" href={_relativeURL( `/assets/images/favicons/favicon-32x32.png`, _ID )} />
		<link rel="icon" type="image/png" sizes="16x16" href={_relativeURL( `/assets/images/favicons/favicon-16x16.png`, _ID )} />
		<link rel="manifest" href={_relativeURL( `/assets/images/favicons/site.webmanifest`, _ID )} />

		<noscript>
			<link rel="stylesheet" href={_relativeURL( `/assets/css/noscript.css`, _ID )} />
		</noscript>
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
			{_ID == 'index' ? <IndexHeader title={title} _parents={_parents} _ID={_ID} _pages={_pages} _nav={_nav} _globalProp={_globalProp} />
		        : _ID == 'cv' ? <ResumeHeader title={title} _ID={_ID} _pages={_pages} _nav={_nav} />
		        : <Header title={title} _parents={_parents} _ID={_ID} _pages={_pages} _nav={_nav} _globalProp={_globalProp} />
		      }
		    {_ID != 'index' ? <div className="section works">
				<div className="content">
					{ main }
				</div>
			</div>: null}

			{
				script != undefined
					? ( <script src={ _relativeURL( `/assets/js/${ script }.js`, _ID ) } /> )
					: null
			}

			<footer>
				<div className="soc">
					<a target="_blank" href="https://infosec.exchange/@sudorandom"><ion-icon name="logo-mastodon"></ion-icon></a>
					<a target="_blank" href="https://twitter.com/sudorandom"><ion-icon name="logo-twitter"></ion-icon></a>
					<a target="_blank" href="https://github.com/sudorandom"><ion-icon name="logo-github"></ion-icon></a>
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
	    <script src={_relativeURL( `/assets/js/imagesloaded.pkgd.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/isotope.pkgd.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/glitche-scripts.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/jqphotoswipe/photoswipe.min.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/jqphotoswipe/photoswipe-ui-default.min.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/jqphotoswipe/jqPhotoSwipe.min.js`, _ID )}></script>
	    <script src={_relativeURL( `/assets/js/magnific-popup.js`, _ID )}></script>
	    <a rel="me" href="https://infosec.exchange/@sudorandom"></a>
		<script type="module" src="https://unpkg.com/ionicons@7.1.0/dist/ionicons/ionicons.esm.js"></script>
		<script noModule src="https://unpkg.com/ionicons@7.1.0/dist/ionicons/ionicons.js"></script>
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
