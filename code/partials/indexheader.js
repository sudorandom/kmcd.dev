import PropTypes from 'prop-types';
import React, { Fragment } from 'react';

import Nav from './Nav';

/**
 * The IndexHeader component
 *
 * @disable-docs
 */
const IndexHeader = ({ title, _parents, _ID, _pages, _nav, _globalProp }) => (
	<Fragment>
		<Nav _ID={_ID} _pages={_pages} _nav={_nav} />
		<div className="wrapper">
			<div className="section started fullheight">
				<div className="centrize full-width">
					<div className="vertical-center">
						<div className="started-content">
							<div className="h-title glitch-effect" data-text={ _globalProp['sitename'] }>{ _globalProp['sitename'] }</div>
							<div className="h-subtitle typing-subtitle">
								<p>Welcome to my personal website</p>
								<p>Personal websites aren't that popular anymore</p>
								<p>But I think it is important to have a space all to yourself</p>
								<p>Welcome!</p>
								<p>...</p>
								<p>...</p>
								<p>Why are you still reading this?</p>
								<p>...</p>
								<p>...</p>
								<p>Go away. The nav is on the top left.</p>
								<p>Browse around. Look at some of my projects.</p>
								<p>...</p>
								<p>Please?</p>
								<p>...</p>
								<p>...</p>
								<p>...</p>
								<p>01100111 01101111 00100000 01100001 01110111 01100001 01111001</p>
								<p>01100111 01101111 00100000 01100001 01110111 01100001 01111001</p>
								<p>01100111 01101111 00100000 01100001 01110111 01100001 01111001</p>
								<p>01000111 01001111 00100000 01000001 01010111 01000001 01011001</p>
							</div>
							<span className="typed-subtitle"></span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Fragment>
);

IndexHeader.defaultProps = {};

export default IndexHeader;
