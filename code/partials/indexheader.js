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
			<div className="section started">
				<div className="centrize full-width">
					<div className="vertical-center">
						<div className="started-content">
							<div className="h-title glitch-effect" data-text={ _globalProp['sitename'] }>{ _globalProp['sitename'] }</div>
							<div className="h-subtitle typing-subtitle">
								<p>Welcome to my personal website</p>
								<p>These aren't that popular anymore</p>
								<p>But I think it is important to have a space all to yourself</p>
								<p>Welcome!</p>
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
